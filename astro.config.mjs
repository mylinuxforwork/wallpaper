import { defineConfig } from 'astro/config';
import tailwind from '@astrojs/tailwind';

const owner = process.env.GITHUB_REPOSITORY_OWNER || 'rootagi';

// https://astro.build/config
export default defineConfig({
  site: `https://${owner}.github.io`,
  base: '/wallpaper',
  integrations: [tailwind()],
  build: {
    format: 'directory'
  }
});
