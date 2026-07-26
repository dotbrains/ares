'use client';

import { Cloud, ServerCog, Workflow } from 'lucide-react';

export function UseCasesSection() {
  const useCases = [
    {
      icon: <Cloud className="w-6 h-6" />,
      title: 'Fresh VPS setup',
      description: 'Run a repeatable hardening baseline after provisioning a new public Linux server.',
    },
    {
      icon: <ServerCog className="w-6 h-6" />,
      title: 'Web hosts',
      description: 'Use the web profile to preserve SSH while allowing inbound HTTP and HTTPS.',
    },
    {
      icon: <Workflow className="w-6 h-6" />,
      title: 'Team baselines',
      description: 'Keep shared defaults in config and add explicit local plugins for site-specific needs.',
    },
  ];

  return (
    <section id="use-cases" className="py-12 sm:py-16 lg:py-20 bg-dark-slate">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">Use Cases</h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            Common workflows for small VPS fleets and one-off servers
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
