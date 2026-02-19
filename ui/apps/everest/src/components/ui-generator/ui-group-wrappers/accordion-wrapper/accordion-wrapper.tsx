import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Typography,
} from '@mui/material';
import { AccordionWrapperProps } from './accordion-wrapper.types';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';

// TODO can be customizable with props
const AccordionWrapper = ({ children, label }: AccordionWrapperProps) => {
  return (
    <Accordion defaultExpanded>
      {label && (
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Box display="flex" alignItems="center">
            {/* {hasError && (
                            <ErrorOutlineIcon
                                color="error"
                                sx={{ mr: 1, position: 'relative', bottom: 1 }}
                            />
                        )} */}
            <Typography variant="sectionHeading" textTransform="capitalize">
              {label}
            </Typography>
          </Box>
        </AccordionSummary>
      )}
      <AccordionDetails style={{ display: 'flex', flexDirection: 'column' }}>
        {children}
      </AccordionDetails>
    </Accordion>
  );
};

export default AccordionWrapper;
