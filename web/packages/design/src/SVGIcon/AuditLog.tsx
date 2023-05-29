/*
Copyright 2023 Gravitational, Inc.

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

import React from 'react';

import { SVGIcon } from './SVGIcon';

import type { SVGIconProps } from './common';

export function AuditLogIcon({ size = 14, fill }: SVGIconProps) {
  return (
    <SVGIcon viewBox="0 0 14 10" size={size} fill={fill}>
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M0 1.84993C0 2.42874 0.471188 2.89993 1.05 2.89993C1.62881 2.89993 2.1 2.42874 2.1 1.84993C2.1 1.27111 1.62881 0.799927 1.05 0.799927C0.471188 0.799927 0 1.27111 0 1.84993ZM0.7 1.84993C0.7 1.65655 0.856625 1.49993 1.05 1.49993C1.24338 1.49993 1.4 1.65655 1.4 1.84993C1.4 2.0433 1.24338 2.19993 1.05 2.19993C0.856625 2.19993 0.7 2.0433 0.7 1.84993ZM13.6502 2.19997H3.15017C2.9568 2.19997 2.80017 2.04334 2.80017 1.84997C2.80017 1.65659 2.9568 1.49997 3.15017 1.49997H13.6502C13.8435 1.49997 14.0002 1.65659 14.0002 1.84997C14.0002 2.04334 13.8435 2.19997 13.6502 2.19997ZM3.15017 5.69997H13.6502C13.8435 5.69997 14.0002 5.54334 14.0002 5.34997C14.0002 5.15659 13.8435 4.99997 13.6502 4.99997H3.15017C2.9568 4.99997 2.80017 5.15659 2.80017 5.34997C2.80017 5.54334 2.9568 5.69997 3.15017 5.69997ZM3.15017 9.19997H13.6502C13.8435 9.19997 14.0002 9.04334 14.0002 8.84997C14.0002 8.65659 13.8435 8.49997 13.6502 8.49997H3.15017C2.9568 8.49997 2.80017 8.65659 2.80017 8.84997C2.80017 9.04334 2.9568 9.19997 3.15017 9.19997ZM1.05 6.39993C0.471188 6.39993 0 5.92874 0 5.34993C0 4.77111 0.471188 4.29993 1.05 4.29993C1.62881 4.29993 2.1 4.77111 2.1 5.34993C2.1 5.92874 1.62881 6.39993 1.05 6.39993ZM1.05 4.99993C0.856625 4.99993 0.7 5.15655 0.7 5.34993C0.7 5.5433 0.856625 5.69993 1.05 5.69993C1.24338 5.69993 1.4 5.5433 1.4 5.34993C1.4 5.15655 1.24338 4.99993 1.05 4.99993ZM0 8.84993C0 9.42874 0.471188 9.89993 1.05 9.89993C1.62881 9.89993 2.1 9.42874 2.1 8.84993C2.1 8.27111 1.62881 7.79993 1.05 7.79993C0.471188 7.79993 0 8.27111 0 8.84993ZM0.7 8.84993C0.7 8.65655 0.856625 8.49993 1.05 8.49993C1.24338 8.49993 1.4 8.65655 1.4 8.84993C1.4 9.0433 1.24338 9.19993 1.05 9.19993C0.856625 9.19993 0.7 9.0433 0.7 8.84993Z"
      />
    </SVGIcon>
  );
}
