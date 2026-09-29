// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { Box } from '@mui/material';
import { buildHighlightRegExp } from '../card-picker.utils';

interface HighlightProps {
  text: string;
  tokens: string[];
}

// Wraps every occurrence of any token in <mark>, case-insensitively.
export const Highlight = ({ text, tokens }: HighlightProps) => {
  if (tokens.length === 0 || !text) {
    return <>{text}</>;
  }
  const parts = text.split(buildHighlightRegExp(tokens));
  return (
    <>
      {parts.map((part, i) =>
        tokens.includes(part.toLowerCase()) ? (
          <Box
            key={i}
            component="mark"
            sx={{
              px: 0.25,
              borderRadius: 0.5,
              bgcolor: 'warning.main',
              color: 'warning.contrastText',
            }}
          >
            {part}
          </Box>
        ) : (
          <span key={i}>{part}</span>
        )
      )}
    </>
  );
};
