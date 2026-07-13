import ArrowBackIcon from '@mui/icons-material/ArrowBack';

import { Box, Typography } from '@mui/material';
import { Link } from 'react-router-dom';
import { messages } from '../../policies.messages';

interface BackToProps {
  to: string;
  prevPage: string;
}

const BackTo = ({ to, prevPage }: BackToProps) => {
  return (
    <Link to={to} style={{ textDecoration: 'none' }}>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'start',
          alignItems: 'center',
          gap: 1,
          mb: 2,
        }}
      >
        <ArrowBackIcon sx={{ color: 'primary.main' }} />
        <Typography variant="body2" sx={{
          color: "primary.main"
        }}>
          {messages.backTo.back} {prevPage}
        </Typography>
      </Box>
    </Link>
  );
};
export default BackTo;