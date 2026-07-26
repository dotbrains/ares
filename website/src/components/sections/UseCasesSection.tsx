'use client';

import { Terminal, Settings, Zap } from 'lucide-react';

export function UseCasesSection() {
  // TODO: Replace with your project's use cases.
  const useCases = [
    {
      icon: <Terminal className="w-6 h-6" />,
      title: 'Use Case One',
      description: 'Describe who uses this and what problem it solves for them.',
    },
    {
      icon: <Settings className="w-6 h-6" />,
      title: 'Use Case Two',
      description: 'Describe the second use case.',
    },
    {
      icon: <Zap className="w-6 h-6" />,
      title: 'Use Case Three',
      description: 'Describe the third use case.',
    },
  ];

  return (
    <section id="use-cases" className="py-12 sm:py-16 lg:py-20 bg-dark-slate">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">Use Cases</h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            ares adapts to how your team works
          </p>
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6 lg:gap-8">
          {useCases.map((useCase, index) => (
            <div key={index} className="bg-dark-gray/50 border border-accent-primary/20 rounded-xl p-5 sm:p-6 hover:border-accent-secondary/40 transition-all">
              <div className="w-10 h-10 sm:w-12 sm:h-12 bg-gradient-to-br from-accent-primary to-accent-secondary rounded-lg flex items-center justify-center text-white mb-3 sm:mb-4">
                {useCase.icon}
              </div>
              <h3 className="text-lg sm:text-xl font-semibold text-cream mb-2">{useCase.title}</h3>
              <p className="text-cream/60 text-sm sm:text-base leading-relaxed">{useCase.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
