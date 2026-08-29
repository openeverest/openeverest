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

import { useState, useMemo, useCallback, useEffect } from 'react';
// MUI's ThemeProvider (not emotion's) is required so cssVariables injects the
// `--mui-*` custom properties; it also provides the emotion theme context.
import { PaletteMode, createTheme, ThemeProvider } from '@mui/material';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeContextProviderProps } from './theme-context-provider.types';
import { ColorModeContext } from './theme-contexts';

const COLOR_MODE_STORAGE_KEY = 'colorMode';

const getColorModeFromLocalStorage = (): PaletteMode => {
  const colorMode = localStorage.getItem(COLOR_MODE_STORAGE_KEY);

  if (colorMode && (colorMode === 'light' || colorMode === 'dark')) {
    return colorMode;
  }

  return 'light';
};

const ThemeContextProvider = ({
  children,
  themeOptions,
  saveColorModeOnLocalStorage,
}: ThemeContextProviderProps) => {
  const [colorMode, setColorMode] = useState<PaletteMode>(
    saveColorModeOnLocalStorage ? getColorModeFromLocalStorage() : 'light'
  );
  const toggleColorMode = useCallback(() => {
    setColorMode((prevMode) => {
      const newColorMode = prevMode === 'light' ? 'dark' : 'light';
      if (saveColorModeOnLocalStorage) {
        localStorage.setItem(COLOR_MODE_STORAGE_KEY, newColorMode);
      }
      return newColorMode;
    });
  }, [saveColorModeOnLocalStorage]);

  const theme = useMemo(
    // cssVariables emits the palette/spacing/typography as `--mui-*` custom
    // properties on :root so plugins with their own MUI can inherit host tokens.
    () => createTheme({ ...themeOptions(colorMode), cssVariables: true }),
    [colorMode, themeOptions]
  );

  // Published so plugins (which bundle their own MUI) can observe the active
  // color scheme and re-render, since they can't read the host's React context.
  useEffect(() => {
    document.documentElement.setAttribute(
      'data-everest-color-scheme',
      colorMode
    );
  }, [colorMode]);

  return (
    <ColorModeContext.Provider value={{ colorMode, toggleColorMode }}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ColorModeContext.Provider>
  );
};

export default ThemeContextProvider;
