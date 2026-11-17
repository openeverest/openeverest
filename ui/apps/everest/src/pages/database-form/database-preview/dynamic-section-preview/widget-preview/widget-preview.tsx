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

import { Stack, Typography } from '@mui/material';
import { ShowMoreDialog } from '@percona/ui-lib';
import { WidgetComponent } from 'components/ui-generator/ui-generator.types';
import { WidgetSummary } from 'components/ui-generator/widget-summary-registry';
import { PreviewContentText } from '../../preview-section';

interface WidgetPreviewProps {
  label: string;
  item: WidgetComponent;
  value: unknown;
  summary: WidgetSummary;
}

// A widget's digest lines, with its full read-only view one click away.
export const WidgetPreview = ({
  label,
  item,
  value,
  summary,
}: WidgetPreviewProps) => {
  const rows = summary.digest(value);

  if (rows.length === 0) {
    return <PreviewContentText text={`${label}: -`} />;
  }

  return (
    <Stack
      spacing={0.25}
      data-testid="preview-widget"
      sx={{ alignItems: 'flex-start', width: '100%' }}
    >
      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
        {label}:
      </Typography>
      {rows.map((row) => (
        <PreviewContentText
          key={row.label}
          text={`${row.label}: ${row.text}`}
        />
      ))}
      <ShowMoreDialog dialogTitle={label} dataTestId="preview-widget">
        <summary.View item={item} value={value} />
      </ShowMoreDialog>
    </Stack>
  );
};
