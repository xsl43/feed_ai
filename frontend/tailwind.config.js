/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        weibo: {
          bg: '#FAFAFA',
          card: '#FFFFFF',
          primary: '#E6162D',
          'primary-hover': '#C8102E',
          text: '#333333',
          'text-secondary': '#999999',
          'text-muted': '#BBBBBB',
          border: '#F0F0F0',
          'border-strong': '#E0E0E0',
          highlight: '#FFF5F5',
          link: '#EB7340',
          gold: '#FF8200',
          silver: '#8B96A0',
          bronze: '#CD7E63',
        }
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'Roboto', '"Helvetica Neue"', 'Arial', '"PingFang SC"', '"Microsoft YaHei"', 'sans-serif'],
      },
      fontSize: {
        '2xs': '0.625rem',
      }
    },
  },
  plugins: [],
}
