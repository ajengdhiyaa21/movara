/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './web/templates/**/*.html',
    './web/static/**/*.js',
    './internal/**/*.go',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'monospace'],
      },
    },
  },
  plugins: [],
};
