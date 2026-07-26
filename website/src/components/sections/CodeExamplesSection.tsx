'use client';

import React, { useState } from 'react';
import { CodeBlock } from '@/components/CodeBlock';

export function CodeExamplesSection() {
  const [activeTab, setActiveTab] = useState<'basic' | 'config' | 'plugins'>('basic');

  const examples = {
    basic: `# Inspect first
$ sudo ares --dry-run

# Apply after review
$ sudo ares --yes

# Use the web profile
$ sudo ares --profile web --yes`,
    config: `# ~/.config/ares/config.yaml
profile: basic
plugins:
  enabled:
    - ssh-hardening
    - firewall-auto
    - fail2ban
    - security-updates
    - sysctl-baseline`,
    plugins: `# Browse built-ins
$ ares plugins list
$ ares plugins show ssh-hardening
$ ares plugins snippet strict-profile`,
  };

  const tabs = [
    { key: 'basic' as const, label: 'Basic Usage', language: 'bash' },
    { key: 'config' as const, label: 'Config', language: 'yaml' },
    { key: 'plugins' as const, label: 'Plugins', language: 'bash' },
  ];

  return (
    <section id="code-examples" className="py-12 sm:py-16 lg:py-20 bg-dark-gray/50">
      <div className="max-w-6xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">Code Examples</h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            See ares in action
          </p>
        </div>
        <div className="bg-dark-slate border border-accent-primary/30 rounded-xl overflow-hidden">
          <div className="flex border-b border-accent-primary/30 overflow-x-auto">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex-1 px-3 sm:px-6 py-3 sm:py-4 text-xs sm:text-sm font-semibold transition-colors whitespace-nowrap ${
                  activeTab === tab.key
                    ? 'bg-dark-gray/50 text-accent-primary border-b-2 border-accent-primary'
                    : 'text-cream/70 hover:text-cream hover:bg-dark-gray/30'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
          <div className="p-4 sm:p-6 overflow-x-auto">
            <CodeBlock
              code={examples[activeTab]}
              language={tabs.find((t) => t.key === activeTab)?.language}
            />
          </div>
        </div>
      </div>
    </section>
  );
}
