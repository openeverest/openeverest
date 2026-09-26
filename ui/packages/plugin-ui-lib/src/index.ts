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

export {
  PluginThemeProvider,
  useHostColorMode,
} from './plugin-theme-provider';
export type { PluginThemeProviderProps } from './plugin-theme-provider';

// Curated re-exports of host-themed MUI primitives. Plugins import these
// instead of `@mui/material` directly so the stable surface can be governed
// over time. React stays external (host singleton via the import map).
export {
  Box,
  Stack,
  Grid,
  Container,
  Paper,
  Card,
  CardContent,
  CardHeader,
  CardActions,
  Typography,
  Button,
  IconButton,
  Link,
  TextField,
  InputBase,
  InputAdornment,
  Select,
  MenuItem,
  FormControl,
  FormControlLabel,
  InputLabel,
  Checkbox,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  Drawer,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Divider,
  Chip,
  Tooltip,
  Alert,
  AlertTitle,
  CircularProgress,
  LinearProgress,
  Skeleton,
  Menu,
  List,
  ListItem,
  ListItemText,
  ListItemButton,
  ListItemIcon,
  Collapse,
  Tabs,
  Tab,
  Portal,
  Snackbar,
  Avatar,
  useTheme,
  useMediaQuery,
  styled,
  alpha,
} from '@mui/material';

export type { SxProps, Theme } from '@mui/material';
