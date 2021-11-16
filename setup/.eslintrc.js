module.exports = {
  "env": {
    "commonjs": true,
    "es6": true,
    "jest": true,
    "node": true
  },
  extends: [
    'airbnb-base',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking',
    'prettier',
  ],
  parserOptions: {
    project: './tsconfig.json',
  },

  parser: '@typescript-eslint/parser',
  plugins: ['@typescript-eslint'],
  settings: {
    'import/resolver': {
      node: {
        paths: ['src'],
        extensions: ['.ts', '.js'],
      },
    },
  },
  rules: {
    'import/extensions': 'off',
    'import/no-extraneous-dependencies': ['error', {'devDependencies': ['**/*.test.ts']}],
    'max-classes-per-file': 'off',
    'max-len': 'off',
    'no-await-in-loop': 'off',
    'no-continue': 'off',
    'no-plusplus': 'off',
    'no-restricted-syntax': 'off',
    'no-shadow': 'off',
    'no-use-before-define': 'off',
    'no-useless-constructor': 'off',
  },
  ignorePatterns: [
    'src/generated/**/*',
    'dist/**/*',
    '.eslintrc.js',
  ],
};
