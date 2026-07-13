import { Box, BoxProps, Stack, Typography } from '@mui/material';

type Props = {
  title?: React.ReactNode;
  children: React.ReactNode;
  boxProps?: BoxProps;
};

const RoundedBox = ({ title, children, boxProps }: Props) => {
  const { sx, ...restProps } = boxProps || {};

  return (
    <Box
      className="percona-rounded-box"
      sx={theme => ({
        p: 2,
        borderStyle: 'solid',
        borderWidth: '1px',
        borderColor: theme.palette.divider,
        borderRadius: 2,
        ...sx
      })}
      {...restProps}
    >
      <Stack>
        {title &&
          (typeof title === 'string' ? (
            <Typography variant="sectionHeading">{title}</Typography>
          ) : (
            title
          ))}
        <Box>{children}</Box>
      </Stack>
    </Box>
  );
};

export default RoundedBox;
