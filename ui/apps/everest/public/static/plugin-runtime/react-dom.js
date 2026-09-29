// Import-map target for the bare `react-dom` specifier in plugin bundles.
// Re-exports the host's single ReactDOM instance (used by MUI portals).
const RD = window.__EVEREST_PLUGIN_RUNTIME__.ReactDOM;

export default RD;

export const {
  createPortal,
  flushSync,
  unstable_batchedUpdates,
  findDOMNode,
  render,
  hydrate,
  unmountComponentAtNode,
  version,
} = RD;
