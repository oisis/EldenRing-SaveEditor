/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'er-dark': '#1a1a1a',
        'er-gold': '#c1a35f',
        'er-gray': '#2d2d2d',
      }
    },
  },
  plugins: [],
}
