import { Box, IconButton, Stack, Typography } from '@mui/material';
import ArrowBackIosIcon from '@mui/icons-material/ArrowBackIos';
import { BackNavigationTextProps } from './back-nativation-text.types';

const BackNavigationText = ({
  text,
  onBackClick,
  stackProps,
  rightSlot,
}: BackNavigationTextProps) => (
  <Stack
    {...stackProps}
    sx={[{
      gap: 1,
      alignItems: "center",
      flexDirection: "row"
    }, ...(stackProps?.sx
      ? (Array.isArray(stackProps.sx) ? stackProps.sx : [stackProps.sx])
      : [])]}>
    <IconButton onClick={onBackClick}>
      <ArrowBackIosIcon sx={{ pl: '10px' }} fontSize="large" />
    </IconButton>
    <Typography variant="h4">{text}</Typography>
    {rightSlot && <Box sx={{ ml: 'auto' }}>{rightSlot}</Box>}
  </Stack>
);

export default BackNavigationText;
