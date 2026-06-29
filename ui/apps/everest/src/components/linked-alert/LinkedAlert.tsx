import { Alert, Box, Link, Stack } from '@mui/material';
import { LinkedAlertProps } from './LinkedAlert.types';

const LinkedAlert = ({
  message,
  linkProps,
  ...alertProps
}: LinkedAlertProps) => {
  const { linkContent } = linkProps;
  return (
    <Alert
      severity="warning"
      sx={{
        '& .MuiAlert-message': {
          width: '100%',
        },
      }}
      {...alertProps}
    >
      <Stack
        sx={{
          flexDirection: 'row',
          justifyContent: 'space-between',
        }}
      >
        <Box>{message}</Box>
        <Box>
          <Link
            sx={{
              fontWeight: 600,
              whiteSpace: 'nowrap',
              ml: 2,
            }}
            underline="none"
            color="inherit"
            target="_blank"
            rel="noopener noreferrer"
            {...linkProps}
          >
            {linkContent}
          </Link>
        </Box>
      </Stack>
    </Alert>
  );
};
export default LinkedAlert;
