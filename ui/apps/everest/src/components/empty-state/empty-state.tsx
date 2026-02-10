import {
  Button,
  Divider,
  Typography,
  Link,
  Stack,
  ButtonProps,
} from '@mui/material';
import HelpIcon from '@mui/icons-material/HelpOutlineOutlined';
import { EmptyStateIcon } from '@percona/ui-lib';

const ContactSupportLink = ({ msg }: { msg: string }) => {
  return (
    <Link target="_blank" rel="noopener" href="https://openeverest.io/support/">
      <Button
        startIcon={
          <HelpIcon
            sx={{
              borderRadius: '10px',
            }}
          />
        }
      >
        {msg}
      </Button>
    </Link>
  );
};

type EmptyStateProps = {
  showCreationButton?: boolean;
  contentSlot?: React.ReactNode;
  buttonSlot?: React.ReactNode;
  buttonProps?: ButtonProps;
  buttonText?: string;
  onButtonClick?: () => void;
};

const EmptyState = ({
  showCreationButton = true,
  contentSlot,
  buttonSlot,
  buttonProps,
  buttonText,
  onButtonClick = () => {},
}: EmptyStateProps) => {
  return (
    <>
      <Stack
        alignItems="center"
        sx={{
          flexDirection: 'column',
          backgroundColor: (theme) =>
            theme.palette.surfaces?.elevation0 || 'transparent',
          p: 3,
          gap: 2,
        }}
      >
        <EmptyStateIcon w="60px" h="60px" />
        <Stack alignItems="center">
          {contentSlot ? contentSlot : <Typography>No data to show</Typography>}
        </Stack>
        {buttonSlot
          ? buttonSlot
          : showCreationButton && (
              <Button
                variant="contained"
                onClick={onButtonClick}
                {...buttonProps}
              >
                {buttonText || 'Create'}
              </Button>
            )}

        <Divider sx={{ width: '30%', marginTop: '10px' }} />
        <ContactSupportLink msg="Get support" />
      </Stack>
    </>
  );
};

export default EmptyState;
