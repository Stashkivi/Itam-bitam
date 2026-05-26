import type { Config } from 'tailwindcss';

const config: Config = {
  content:  ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        surface:  '#0f172a',  // graph canvas background
        card:     '#1e293b',  // node card background
        cardHover:'#263348',
        border:   '#334155',
        muted:    '#64748b',
      },
      keyframes: {
        'pulse-ring': {
          '0%':   { boxShadow: '0 0 0 0px var(--ring-color)' },
          '70%':  { boxShadow: '0 0 0 12px transparent' },
          '100%': { boxShadow: '0 0 0 0px transparent' },
        },
        'slide-in': {
          from: { transform: 'translateX(120%)' },
          to:   { transform: 'translateX(0)' },
        },
        'fade-out': {
          from: { opacity: '1' },
          to:   { opacity: '0' },
        },
      },
      animation: {
        'pulse-ring':  'pulse-ring 1.8s ease-in-out infinite',
        'slide-in':    'slide-in 0.3s ease-out',
        'fade-out':    'fade-out 0.4s ease-in forwards',
      },
    },
  },
  plugins: [],
};

export default config;
