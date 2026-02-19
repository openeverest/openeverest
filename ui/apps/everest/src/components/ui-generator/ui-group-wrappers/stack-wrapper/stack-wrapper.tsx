import { Stack } from '@mui/material';
import { StackWrapperProps } from './stack-wrapper.types';

const StackWrapper = ({ children }: StackWrapperProps) => {
  return (
    <Stack
      direction="row"
      spacing={2}
      alignItems="flex-start"
      sx={{
        width: '100%',
        flexWrap: 'nowrap',
        '> *': { flex: '1 1 0', minWidth: 0 },
      }}
    >
      {children}
    </Stack>
  );
};

export default StackWrapper;
