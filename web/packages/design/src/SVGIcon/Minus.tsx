/*
 * Copyright 2023 Gravitational, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';

import { SVGIconProps } from 'design/SVGIcon/common';
import { SVGIcon } from 'design/SVGIcon/SVGIcon';

export function MinusIcon({ size = 10, fill }: SVGIconProps) {
  return (
    <SVGIcon viewBox="0 0 10 10" size={size} fill={fill}>
      <path d="M9.07388 5.574H0.925293V4.42542H9.07388V5.574Z" />
    </SVGIcon>
  );
}
