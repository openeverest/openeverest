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

import { Button } from '@mui/material';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import { Messages } from './configure-more.messages';

export const ConfigureMore = ({ onClick }: { onClick: () => void }) => {
  return (
    <Button type="button" onClick={onClick}>
      {Messages.configMore}
      <EditOutlinedIcon
        sx={{
          verticalAlign: 'text-bottom',
          pl: 1,
        }}
        fontSize="medium"
        data-testid={`config-more-button`}
      />
    </Button>
  );
};

export default ConfigureMore;
