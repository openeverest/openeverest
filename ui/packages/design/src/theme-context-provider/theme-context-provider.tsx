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
import { ThemeProvider } from '@emotion/react';
import { PaletteMode, createTheme } from '@mui/material';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeContextProviderProps } from './theme-context-provider.types';
import { ColorModeContext } from './theme-contexts';
import { writeEverestTokens } from './everest-tokens';

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
    () => createTheme(themeOptions(colorMode)),
    [colorMode, themeOptions]
  );

  // Publish the host design tokens as `--everest-*` CSS variables and the active
  // color scheme so plugins (which bundle their own MUI and can't read the host
  // React context) inherit the design system and re-render on dark-mode toggle.
  useEffect(() => {
    writeEverestTokens(theme);
    document.documentElement.setAttribute(
      'data-everest-color-scheme',
      colorMode
    );
  }, [theme, colorMode]);

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
