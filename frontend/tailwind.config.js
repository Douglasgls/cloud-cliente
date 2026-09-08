/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#f0f7fe',
          100: '#dff0fc',
          200: '#b9def9',
          300: '#87c6f5',
          400: '#3387d6',
          500: '#0066B2',
          600: '#005290',
          700: '#004b85',
          800: '#003e6f',
          900: '#00345e',
          accent: '#0077D9',
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
      }
    },
  },
  plugins: [],
}
