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
    <Box
      sx={{
        display: "flex",
        ml: "auto",
        alignItems: "center"
      }}>
      {children}
      <Tooltip title={tooltip} placement="right" arrow>
        <span>
          <IconButton
            onClick={onClick}
            disabled={disabled}
            sx={[disabled ? {
              opacity: 0.5
            } : {
              opacity: 1
            }, disabled ? {
              cursor: 'not-allowed'
            } : {
              cursor: 'pointer'
            }]}
          >
            <InfoIcon
              sx={[{
                width: '20px'
              }, disabled ? {
                color: 'GrayText'
              } : {
                color: 'default'
              }]}
            />
          </IconButton>
        </span>
      </Tooltip>
    </Box>
  );
};

export default WithInfoIcon;
