// everest
// Copyright (C) 2023 Percona LLC
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
import StorageIcon from '@mui/icons-material/Storage';
import SettingsIcon from '@mui/icons-material/Settings';
import SchemaIcon from '@mui/icons-material/Schema';
import { EverestRoute } from './Drawer.types';
import { SettingsTabs } from 'pages/settings/settings.types';

export const DRAWER_WIDTH = 200;

export const ROUTES: EverestRoute[] = [
  {
    to: '/databases',
    icon: StorageIcon,
    text: 'Databases',
  },
  {
    to: `settings/${SettingsTabs.storageLocations}`,
    icon: SettingsIcon,
    text: 'Settings',
  },
  //TODO ui-generator builder should be hided from ui and delivered separately with tests coverage
  {
    to: '/ui-generator-builder',
    icon: SchemaIcon,
    text: 'UI Generator Builder',
  },
];
