const importPlugin = require("eslint-plugin-import");
const tsParser = require("@typescript-eslint/parser");

module.exports = [
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      parser: tsParser,
    },
    plugins: {
      import: importPlugin,
    },
    settings: {
      "import/resolver": {
        node: {
          extensions: [".js", ".jsx", ".ts", ".tsx"],
        },
      },
    },
    rules: {
      "import/no-restricted-paths": [
        "error",
        {
          zones: [
            {
              target: "./src/domain",
              from: "./src/features",
              message: "domain は features に依存できません",
            },
          ],
        },
      ],
    },
  },
];
