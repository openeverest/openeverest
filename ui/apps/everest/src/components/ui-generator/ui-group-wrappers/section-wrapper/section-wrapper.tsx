import { Box, Typography } from '@mui/material';
import RoundedBox from 'components/rounded-box';
import { SectionWrapperProps } from './section-wrapper.types';

const SectionWrapper = ({
  label,
  description,
  disabled = false,
  children,
}: SectionWrapperProps) => {
  return (
    <RoundedBox
      title={label}
      boxProps={{
        sx: {
          opacity: disabled ? 0.5 : 1,
          pointerEvents: disabled ? 'none' : undefined,
        },
      }}
    >
      {description && (
        <Typography
          variant="caption"
          color="text.secondary"
          display="block"
          sx={{ mt: 0.5, mb: 1.5 }}
        >
          {description}
        </Typography>
      )}
      <Box>{children}</Box>
    </RoundedBox>
  );
};

export default SectionWrapper;
