'use client';

import { Terminal, Settings, Zap } from 'lucide-react';

export function FeaturesSection() {
  // TODO: Replace with your project's features and icons.
  const features = [
    {
      icon: <Terminal className="w-6 h-6" />,
      title: 'Feature One',
      description: 'Describe your first key feature here. What does it do? Why does the user care?',
    },
    {
      icon: <Settings className="w-6 h-6" />,
      title: 'Feature Two',
      description: 'Describe your second key feature here.',
    },
    {
      icon: <Zap className="w-6 h-6" />,
      title: 'Feature Three',
      description: 'Describe your third key feature here.',
    },
  ];

  return (
    <section id="features" className="py-12 sm:py-16 lg:py-20 bg-dark-slate">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">
            Built for Real Engineering Workflows
          </h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            ares fits into how you already work
          </p>
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6 lg:gap-8">
          {features.map((feature, index) => (
            <div
              key={index}
              className="group bg-dark-gray/50 border border-accent-primary/20 hover:border-accent-secondary/40 rounded-xl p-5 sm:p-6 transition-all hover:shadow-lg hover:shadow-accent-primary/10"
            >
              <div className="w-10 h-10 sm:w-12 sm:h-12 bg-gradient-to-br from-accent-primary to-accent-secondary rounded-lg flex items-center justify-center text-white mb-3 sm:mb-4 group-hover:scale-110 transition-transform">
                {feature.icon}
              </div>
              <h3 className="text-lg sm:text-xl font-semibold text-cream mb-2">{feature.title}</h3>
              <p className="text-cream/60 text-sm sm:text-base leading-relaxed">{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
