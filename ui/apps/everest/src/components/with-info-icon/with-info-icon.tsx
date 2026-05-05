import { Box, IconButton, Tooltip } from '@mui/material';
import InfoIcon from '@mui/icons-material/InfoOutlined';

const WithInfoIcon = ({
  children,
  onClick,
  disabled = false,
  tooltip,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  tooltip: string;
}) => {
  return (
    <Box display="flex" ml="auto" alignItems="center">
      {children}
      <Tooltip title={tooltip} placement="right" arrow>
        <span>
          <IconButton
            onClick={onClick}
            disabled={disabled}
            sx={{
              opacity: disabled ? 0.5 : 1,
              cursor: disabled ? 'not-allowed' : 'pointer',
            }}
          >
            <InfoIcon
              sx={{
                width: '20px',
                color: disabled ? 'GrayText' : 'default',
              }}
            />
          </IconButton>
        </span>
      </Tooltip>
    </Box>
  );
};

export default WithInfoIcon;
