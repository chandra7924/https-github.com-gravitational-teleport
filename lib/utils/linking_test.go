/*
Copyright 2018 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebLinks(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		inResponse *http.Response
		outNext    string
		outPrev    string
		outFirst   string
		outLast    string
	}{
		// 0 - Multiple links in single header. Partial list of relations.
		{
			inResponse: &http.Response{
				Header: http.Header{
					"Link": []string{`<https://api.github.com/user/teams?page=2>; rel="next",
		                              <https://api.github.com/user/teams?page=34>; rel="last"`},
				},
			},
			outNext:  "https://api.github.com/user/teams?page=2",
			outPrev:  "",
			outFirst: "",
			outLast:  "https://api.github.com/user/teams?page=34",
		},
		// 1 - Multiple links in single header. Full list of relations.
		{
			inResponse: &http.Response{
				Header: http.Header{
					"Link": []string{`<https://api.github.com/user/teams?page=2>; rel="next",
		                              <https://api.github.com/user/teams?page=1>; rel="prev",
                                      <https://api.github.com/user/teams?page=1>; rel="first",
		                              <https://api.github.com/user/teams?page=34>; rel="last"`},
				},
			},
			outNext:  "https://api.github.com/user/teams?page=2",
			outPrev:  "https://api.github.com/user/teams?page=1",
			outFirst: "https://api.github.com/user/teams?page=1",
			outLast:  "https://api.github.com/user/teams?page=34",
		},
		// 2 - Multiple links in multiple headers. Full list of relations.
		{
			inResponse: &http.Response{
				Header: http.Header{
					"Link": []string{
						`<https://api.github.com/user/teams?page=1>; rel="next"`,
						`<https://api.github.com/user/teams?page=2>; rel="prev"`,
						`<https://api.github.com/user/teams?page=3>; rel="first"`,
						`<https://api.github.com/user/teams?page=4>; rel="last"`,
					},
				},
			},
			outNext:  "https://api.github.com/user/teams?page=1",
			outPrev:  "https://api.github.com/user/teams?page=2",
			outFirst: "https://api.github.com/user/teams?page=3",
			outLast:  "https://api.github.com/user/teams?page=4",
		},
	}

	for _, tt := range tests {
		wls := ParseWebLinks(tt.inResponse)
		require.Equal(t, wls.NextPage, tt.outNext)
		require.Equal(t, wls.PrevPage, tt.outPrev)
		require.Equal(t, wls.FirstPage, tt.outFirst)
		require.Equal(t, wls.LastPage, tt.outLast)
	}
}
