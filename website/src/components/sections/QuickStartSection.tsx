'use client';

import React, { useState } from 'react';
import { CodeBlock } from '@/components/CodeBlock';

export function QuickStartSection() {
  const [installMethod, setInstallMethod] = useState<'bootstrap' | 'go' | 'release'>('bootstrap');

  const bootstrapExample = `curl -fsSL https://raw.githubusercontent.com/dotbrains/ares/main/install.sh | sudo sh`;
  const goExample = `go install github.com/dotbrains/ares@latest`;

  const releaseExample = `# Linux x86_64
gh release download --repo dotbrains/ares \\
  --pattern 'ares_linux_amd64.tar.gz' --dir /tmp
tar -xzf /tmp/ares_linux_amd64.tar.gz -C /usr/local/bin`;

  const installExamples = { bootstrap: bootstrapExample, go: goExample, release: releaseExample };

  return (
    <section id="quick-start" className="py-12 sm:py-16 lg:py-20 bg-dark-slate overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">Quick Start</h2>
          <p className="text-slate-gray text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            Install ares and get started in under a minute
          </p>
        </div>
        <div className="grid lg:grid-cols-2 gap-8 lg:gap-12 items-start">
          <div className="bg-dark-gray/50 rounded-xl p-6 sm:p-8 border border-accent-primary/20 min-w-0">
            <h3 className="text-xl sm:text-2xl font-bold text-cream mb-4 sm:mb-6">1. Install</h3>
            <div className="flex gap-2 sm:gap-3 mb-6">
              {[
                { key: 'bootstrap' as const, label: 'Bootstrap' },
                { key: 'go' as const, label: 'Go' },
                { key: 'release' as const, label: 'Release' },
              ].map((method) => (
                <button
                  key={method.key}
                  onClick={() => setInstallMethod(method.key)}
                  className={`flex-1 px-3 sm:px-4 py-2.5 rounded-lg text-sm font-semibold transition-all ${
                    installMethod === method.key
                      ? 'bg-gradient-to-r from-accent-primary to-accent-secondary text-white shadow-lg shadow-accent-primary/30'
                      : 'bg-dark-slate text-slate-gray hover:text-cream hover:border-accent-primary/50 border border-accent-primary/30'
                  }`}
                >
                  {method.label}
                </button>
              ))}
            </div>
            <CodeBlock code={installExamples[installMethod]} language="bash" />
          </div>
          <div className="bg-dark-gray/50 rounded-xl p-6 sm:p-8 border border-accent-secondary/20 min-w-0">
            <h3 className="text-xl sm:text-2xl font-bold text-cream mb-4 sm:mb-6">2. Use</h3>
            <CodeBlock
              code={`# Inspect the host and plan
sudo ares --dry-run

# Initialize config
ares config init

# Apply after review
sudo ares --yes`}
              language="bash"
            />
            <div className="mt-6 bg-accent-primary/10 border border-accent-primary/30 rounded-lg p-4 sm:p-5">
              <p className="text-cream text-sm leading-relaxed">
                <span className="text-accent-primary font-semibold">Tip:</span> Use <code className="bg-dark-slate/80 px-2 py-1 rounded text-accent-tertiary font-mono text-xs">ares plan --profile web</code> before opening HTTP and HTTPS.
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
