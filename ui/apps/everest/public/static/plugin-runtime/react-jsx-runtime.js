// Import-map target for `react/jsx-runtime` (automatic JSX) in plugin bundles.
const JSX = window.__EVEREST_PLUGIN_RUNTIME__.ReactJSXRuntime;

export const jsx = JSX.jsx;
export const jsxs = JSX.jsxs;
export const Fragment = JSX.Fragment;
