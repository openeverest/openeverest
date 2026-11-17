import commonjs from "@rollup/plugin-commonjs";
import resolve from "@rollup/plugin-node-resolve";
import typescript from "@rollup/plugin-typescript";
import dts from "rollup-plugin-dts";
import terser from "@rollup/plugin-terser";

// MUI/Emotion/React stay external: the consuming plugin bundles its own pinned
// copies. React is resolved to the host singleton via the host import map.
const external = (id) =>
  /^react($|\/)/.test(id) ||
  /^react-dom($|\/)/.test(id) ||
  /^@mui\//.test(id) ||
  /^@emotion\//.test(id);

export default [
  {
    input: "src/index.ts",
    output: [
      {
        file: "dist/esm/index.js",
        format: "esm",
        sourcemap: true,
      },
    ],
    external,
    plugins: [
      resolve(),
      commonjs(),
      typescript({ tsconfig: "./tsconfig.json" }),
      terser(),
    ],
  },
  {
    input: "dist/esm/typings/index.d.ts",
    output: [{ file: "dist/index.d.ts", format: "esm" }],
    external,
    plugins: [dts()],
  },
];
