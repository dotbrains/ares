'use client';

import { Github, BookOpen, MessageCircle } from 'lucide-react';

export function CTASection() {
  return (
    <section className="py-12 sm:py-16 lg:py-20 bg-gradient-to-br from-accent-primary/10 via-dark-slate to-dark-slate">
      <div className="max-w-5xl mx-auto px-4 sm:px-6 text-center">
        <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-4 sm:mb-6">Ready to Get Started?</h2>
        <p className="text-base sm:text-lg lg:text-xl text-cream/70 mb-8 sm:mb-12 max-w-3xl mx-auto">
          Install ares, inspect the plan, then apply with an explicit confirmation
        </p>
        <div className="grid sm:grid-cols-3 gap-4 sm:gap-6">
          <a href="https://github.com/dotbrains/ares" target="_blank" rel="noopener noreferrer" className="bg-dark-gray/50 border border-accent-primary/30 hover:border-accent-primary rounded-xl p-6 sm:p-8 transition-all group hover:shadow-lg hover:shadow-accent-primary/20">
            <div className="w-12 h-12 sm:w-14 sm:h-14 bg-gradient-to-br from-accent-primary to-accent-secondary rounded-lg flex items-center justify-center mx-auto mb-3 sm:mb-4 group-hover:scale-110 transition-transform">
              <Github className="w-6 h-6 sm:w-7 sm:h-7 text-white" />
            </div>
            <h3 className="text-lg sm:text-xl font-semibold text-cream mb-2">View on GitHub</h3>
            <p className="text-cream/60 text-xs sm:text-sm">Star the repo, fork it, and contribute</p>
          </a>
          <a href="https://github.com/dotbrains/ares/tree/main/docs" target="_blank" rel="noopener noreferrer" className="bg-dark-gray/50 border border-accent-secondary/30 hover:border-accent-secondary rounded-xl p-6 sm:p-8 transition-all group hover:shadow-lg hover:shadow-accent-secondary/20">
            <div className="w-12 h-12 sm:w-14 sm:h-14 bg-gradient-to-br from-accent-secondary to-accent-tertiary rounded-lg flex items-center justify-center mx-auto mb-3 sm:mb-4 group-hover:scale-110 transition-transform">
              <BookOpen className="w-6 h-6 sm:w-7 sm:h-7 text-white" />
            </div>
            <h3 className="text-lg sm:text-xl font-semibold text-cream mb-2">Read the Docs</h3>
            <p className="text-cream/60 text-xs sm:text-sm">Getting started, commands, config, plugins, and recovery</p>
          </a>
          <a href="https://github.com/dotbrains/ares/discussions" target="_blank" rel="noopener noreferrer" className="bg-dark-gray/50 border border-accent-tertiary/30 hover:border-accent-tertiary rounded-xl p-6 sm:p-8 transition-all group hover:shadow-lg hover:shadow-accent-tertiary/20">
            <div className="w-12 h-12 sm:w-14 sm:h-14 bg-gradient-to-br from-accent-tertiary to-accent-primary rounded-lg flex items-center justify-center mx-auto mb-3 sm:mb-4 group-hover:scale-110 transition-transform">
              <MessageCircle className="w-6 h-6 sm:w-7 sm:h-7 text-white" />
            </div>
            <h3 className="text-lg sm:text-xl font-semibold text-cream mb-2">Join Discussion</h3>
            <p className="text-cream/60 text-xs sm:text-sm">Ask questions and share ideas</p>
          </a>
        </div>
      </div>
    </section>
  );
}
