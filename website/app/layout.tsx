import type { Metadata } from 'next';
import '@/styles/globals.css';

export const metadata: Metadata = {
  title: 'ares — Modular VPS hardening runner',
  description: 'ares hardens fresh Linux VPS instances with a safe, modular plugin-based execution model. It detects the host distro, plans changes, preserves SSH access, and applies provider-agnostic security defaults.',
  openGraph: {
    title: 'ares — Modular VPS hardening runner',
    description: 'ares hardens fresh Linux VPS instances with a safe, modular plugin-based execution model. It detects the host distro, plans changes, preserves SSH access, and applies provider-agnostic security defaults.',
    url: 'https://ares.dotbrains.io',
    siteName: 'ares',
    images: [
      {
        url: '/og-image.svg',
        width: 1200,
        height: 630,
        alt: 'ares — Modular VPS hardening runner',
      },
    ],
    locale: 'en_US',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'ares — Modular VPS hardening runner',
    description: 'ares hardens fresh Linux VPS instances with a safe, modular plugin-based execution model. It detects the host distro, plans changes, preserves SSH access, and applies provider-agnostic security defaults.',
    images: ['/og-image.svg'],
  },
  icons: {
    icon: [
      {
        url: '/favicon.svg',
        type: 'image/svg+xml',
      },
    ],
    apple: [
      {
        url: '/favicon.svg',
        type: 'image/svg+xml',
      },
    ],
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <meta charSet="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
      </head>
      <body>{children}</body>
    </html>
  );
}
