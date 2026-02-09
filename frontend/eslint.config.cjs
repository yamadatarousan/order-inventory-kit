const importPlugin = require("eslint-plugin-import");
const tsParser = require("@typescript-eslint/parser");

module.exports = [
  {
    ignores: ["src/api/schema.d.ts"],
  },
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
            {
              target: "./src/domain",
              from: "./src/app",
              message: "domain は app に依存できません",
            },
            {
              target: "./src/domain",
              from: "./src/api",
              message: "domain は api に依存できません",
            },
            {
              target: "./src/features",
              from: "./src/app",
              message: "features は app に依存できません",
            },
            {
              target: "./src/shared",
              from: "./src/domain",
              message: "shared は domain に依存できません",
            },
            {
              target: "./src/shared",
              from: "./src/features",
              message: "shared は features に依存できません",
            },
            {
              target: "./src/shared",
              from: "./src/app",
              message: "shared は app に依存できません",
            },
            {
              target: "./src/shared",
              from: "./src/api",
              message: "shared は api に依存できません",
            },
          ],
        },
      ],
    },
  },
];
