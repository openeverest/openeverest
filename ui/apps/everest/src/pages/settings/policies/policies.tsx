import { Box, Typography } from '@mui/material';
import { FormCard } from 'components/form-card';
import { policies } from './constants';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import { Link } from 'react-router-dom';
import { messages } from './policies.messages';

const Policies = () => {
  return (
    <Box
      sx={{
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      <Typography variant="body2" sx={{
        py: 2
      }}>
        {messages.policiesDescription}
      </Typography>
      {policies.map((policy) => (
        <FormCard
          key={policy.name}
          title={policy.name}
          description={policy.description}
          controlComponent={
            <Link to={policy.redirectUrl} style={{ textDecoration: 'none' }}>
              <Box
                sx={{
                  display: 'flex',
                  justifyContent: 'end',
                  alignItems: 'center',
                  gap: 1,
                }}
              >
                <Typography variant="body2" sx={{
                  color: "primary.main"
                }}>
                  {messages.configure}
                </Typography>
                <ArrowForwardIcon sx={{ color: 'primary.main' }} />
              </Box>
            </Link>
          }
        />
      ))}
    </Box>
  );
};
export default Policies;