package tbot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gravitational/teleport/api/client/proto"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/api/utils/retryutils"
	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/client"
	"github.com/gravitational/teleport/lib/tbot/bot"
	"github.com/gravitational/teleport/lib/tbot/identity"
	"github.com/gravitational/teleport/lib/utils"
	"github.com/gravitational/trace"
	"math"
	"time"
)

const botIdentityRenewalRetryLimit = 7

func (b *Bot) renewBotIdentityLoop(
	ctx context.Context,
	reloadChan <-chan struct{},
) error {
	b.log.Infof(
		"Beginning bot identity renewal loop: ttl=%s interval=%s",
		b.cfg.CertificateTTL,
		b.cfg.RenewalInterval,
	)
	if b.cfg.RenewalInterval > b.cfg.CertificateTTL {
		b.log.Errorf(
			"Certificate TTL (%s) is shorter than the renewal interval (%s). The next renewal is likely to fail.",
			b.cfg.CertificateTTL,
			b.cfg.RenewalInterval,
		)
	}

	// Determine where the bot should write its internal data (renewable cert
	// etc)
	botDestination, err := b.cfg.Storage.GetDestination()
	if err != nil {
		return trace.Wrap(err)
	}

	ticker := time.NewTicker(b.cfg.RenewalInterval)
	jitter := retryutils.NewJitter()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-reloadChan:
		}

		var err error
		for attempt := 1; attempt <= botIdentityRenewalRetryLimit; attempt++ {
			b.log.Infof(
				"Renewing bot identity. Attempt %d of %d.",
				attempt,
				botIdentityRenewalRetryLimit,
			)
			err = b.renewBotIdentity(ctx, botDestination)
			if err == nil {
				break
			}

			if attempt != botIdentityRenewalRetryLimit {
				// exponentially back off with jitter, starting at 1 second.
				backoffTime := time.Second * time.Duration(math.Pow(2, float64(attempt-1)))
				backoffTime = jitter(backoffTime)
				b.log.WithError(err).Errorf(
					"Bot identity renewal attempt %d of %d failed. Retrying after %s.",
					attempt,
					renewalRetryLimit,
					backoffTime,
				)
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(backoffTime):
				}
			}
		}
		if err != nil {
			b.log.WithError(err).Errorf("%d bot identity renewals failed. All retry attempts exhausted. Exiting.", renewalRetryLimit)
			return trace.Wrap(err)
		}
		b.log.Infof("Renewed bot identity. Next bot identity renewal in approximately %s.", b.cfg.RenewalInterval)
	}
}

func (b *Bot) renewBotIdentity(
	ctx context.Context,
	botDestination bot.Destination,
) error {
	// Make sure we can still write to the bot's destination.
	if err := identity.VerifyWrite(botDestination); err != nil {
		return trace.Wrap(err, "Cannot write to destination %s, aborting.", botDestination)
	}

	var newIdentity *identity.Identity
	var err error
	if b.cfg.Onboarding.RenewableJoinMethod() {
		// When using a renewable join method, we use GenerateUserCerts to
		// request a new certificate using our current identity.
		newIdentity, err = b.renewIdentityViaAuth(ctx, b.ident(), b.Client())
		if err != nil {
			return trace.Wrap(err)
		}
	} else {
		// When using the non-renewable join methods, we rejoin each time rather
		// than using certificate renewal.
		newIdentity, err = b.getIdentityFromToken()
		if err != nil {
			return trace.Wrap(err)
		}
	}

	identStr, err := describeTLSIdentity(newIdentity)
	if err != nil {
		return trace.Wrap(err, "Could not describe bot's internal identity at %s", botDestination)
	}

	b.log.WithField("identity", identStr).Info("Fetched new bot identity.")
	if err := identity.SaveIdentity(newIdentity, botDestination, identity.BotKinds()...); err != nil {
		return trace.Wrap(err)
	}

	newClient, err := b.AuthenticatedUserClientFromIdentity(ctx, newIdentity)
	if err != nil {
		return trace.Wrap(err)
	}

	b.setClient(newClient)
	b.setIdent(newIdentity)
	b.log.WithField("identity", identStr).Debug("Bot now using new identity.")

	return nil
}

func (b *Bot) renewIdentityViaAuth(
	ctx context.Context,
	ident *identity.Identity,
	client auth.ClientI,
) (*identity.Identity, error) {
	b.log.Info("Fetching bot identity using existing bot identity.")
	if ident == nil || client == nil {
		return nil, trace.BadParameter("renewIdentityWithAuth must be called with non-nil client and identity")
	}
	// Ask the auth server to generate a new set of certs with a new
	// expiration date.
	certs, err := client.GenerateUserCerts(ctx, proto.UserCertsRequest{
		PublicKey: ident.PublicKeyBytes,
		Username:  ident.X509Cert.Subject.CommonName,
		Expires:   time.Now().Add(b.cfg.CertificateTTL),
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	newIdentity, err := identity.ReadIdentityFromStore(
		ident.Params(),
		certs,
		identity.BotKinds()...,
	)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return newIdentity, nil
}

func (b *Bot) getIdentityFromToken() (*identity.Identity, error) {
	b.log.Info("Fetching bot identity using token.")
	addr, err := utils.ParseAddr(b.cfg.AuthServer)
	if err != nil {
		return nil, trace.Wrap(err, "invalid auth server address %+v", b.cfg.AuthServer)
	}

	tlsPrivateKey, sshPublicKey, tlsPublicKey, err := generateKeys()
	if err != nil {
		return nil, trace.Wrap(err, "unable to generate new keypairs")
	}

	token, err := b.cfg.Onboarding.Token()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	expires := time.Now().Add(b.cfg.CertificateTTL)
	params := auth.RegisterParams{
		Token: token,
		ID: auth.IdentityID{
			Role: types.RoleBot,
		},
		AuthServers:        []utils.NetAddr{*addr},
		PublicTLSKey:       tlsPublicKey,
		PublicSSHKey:       sshPublicKey,
		CAPins:             b.cfg.Onboarding.CAPins,
		CAPath:             b.cfg.Onboarding.CAPath,
		GetHostCredentials: client.HostCredentials,
		JoinMethod:         b.cfg.Onboarding.JoinMethod,
		Expires:            &expires,
		FIPS:               b.cfg.FIPS,
		CipherSuites:       b.cfg.CipherSuites(),
	}
	if params.JoinMethod == types.JoinMethodAzure {
		params.AzureParams = auth.AzureParams{
			ClientID: b.cfg.Onboarding.Azure.ClientID,
		}
	}

	certs, err := auth.Register(params)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	sha := sha256.Sum256([]byte(params.Token))
	tokenHash := hex.EncodeToString(sha[:])
	ident, err := identity.ReadIdentityFromStore(&identity.LoadIdentityParams{
		PrivateKeyBytes: tlsPrivateKey,
		PublicKeyBytes:  sshPublicKey,
		TokenHashBytes:  []byte(tokenHash),
	}, certs, identity.BotKinds()...)
	return ident, trace.Wrap(err)
}