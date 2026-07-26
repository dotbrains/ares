'use client';

import { Download, Terminal, FileText } from 'lucide-react';

export function HowItWorksSection() {
  const steps = [
    {
      icon: <Download className="w-8 h-8" />,
      step: '1',
      title: 'Install ares',
      description: 'Install with the release bootstrap script, go install, or a platform archive from GitHub Releases.',
    },
    {
      icon: <Terminal className="w-8 h-8" />,
      step: '2',
      title: 'Inspect the plan',
      description: 'Run ares --dry-run or ares plan to see host detection, selected plugins, risky actions, and warnings.',
    },
    {
      icon: <FileText className="w-8 h-8" />,
      step: '3',
      title: 'Apply deliberately',
      description: 'Run sudo ares --yes after review. Reports, verification notes, and undo guidance are written under /var/log/ares.',
    },
  ];

  return (
    <section id="how-it-works" className="py-12 sm:py-16 lg:py-20 bg-dark-gray/50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">How It Works</h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            Three steps from install to a guarded hardening run
          </p>
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-8">
          {steps.map((step, index) => (
            <div key={index} className="relative sm:col-span-2 lg:col-span-1 last:sm:col-start-auto last:lg:col-start-auto">
              <div className="bg-dark-slate border border-accent-primary/30 rounded-xl p-6 sm:p-8 text-center hover:border-accent-secondary/40 transition-all h-full">
                <div className="w-14 h-14 sm:w-16 sm:h-16 bg-gradient-to-br from-accent-primary to-accent-secondary rounded-full flex items-center justify-center text-white text-xl sm:text-2xl font-bold mx-auto mb-3 sm:mb-4">
                  {step.step}
                </div>
                <div className="w-10 h-10 sm:w-12 sm:h-12 mx-auto mb-3 sm:mb-4 text-accent-primary flex items-center justify-center">
                  {step.icon}
                </div>
                <h3 className="text-lg sm:text-xl font-semibold text-cream mb-2 sm:mb-3">{step.title}</h3>
                <p className="text-cream/60 text-sm sm:text-base leading-relaxed">{step.description}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
