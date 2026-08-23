/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        ink: '#12110f',
        paper: '#efe6d6',
        cadmium: '#e4572e',
        cyan: '#7ec8c3',
        gold: '#d4a017',
        mist: '#2a2824',
        line: '#3a3732',
      },
      fontFamily: {
        display: ['Fraunces', 'serif'],
        serif: ['"Source Serif 4"', 'serif'],
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
}
