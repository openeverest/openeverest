import { useMediaQuery, useTheme } from '@mui/material';

export const useActiveBreakpoint = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const isDesktop = useMediaQuery(theme.breakpoints.up('lg'));
  const isTablet = useMediaQuery(theme.breakpoints.between('sm', 'lg'));
  const isBelowMd = useMediaQuery(theme.breakpoints.down('md'));

  const activeBreakpoint: 'desktop' | 'tablet' | 'mobile' = isMobile
    ? 'mobile'
    : isDesktop
      ? 'desktop'
      : 'tablet';

  return { activeBreakpoint, isMobile, isDesktop, isTablet, isBelowMd };
};
