// Import-map target for `@mui/material/colors` in plugin bundles.
// MUI's createPalette self-imports this barrel with a bare specifier that plugin
// bundlers can't resolve; the host shares these static color scales instead.
const C = window.__EVEREST_PLUGIN_RUNTIME__.MuiColors;

export default C;

export const {
  common,
  red,
  pink,
  purple,
  deepPurple,
  indigo,
  blue,
  lightBlue,
  cyan,
  teal,
  green,
  lightGreen,
  lime,
  yellow,
  amber,
  orange,
  deepOrange,
  brown,
  grey,
  blueGrey,
} = C;
