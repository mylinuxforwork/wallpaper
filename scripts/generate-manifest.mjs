import fs from 'node:fs';
import path from 'node:path';
import sharp from 'sharp';

const SOURCE_DIR = process.cwd();
const PUBLIC_DIR = path.join(SOURCE_DIR, 'public');
const MANIFEST_PATH = path.join(PUBLIC_DIR, 'manifest.json');

// Keyword-based category classification heuristics
function categorizeFilename(filename) {
  const fn = filename.toLowerCase();

  // 1. Space & Astronomy
  if (anyKeyword(fn, ['space', 'nebula', 'planet', 'astro', 'mars', 'blackhole', 'black-whole', 'galaxy', 'star', 'cosmos', 'orbit', 'alien', 'starlight', 'milkyway', 'vulcan', 'expanse', 'comet', 'moon', 'earth', 'saturn', 'jupiter', 'sun', 'solar'])) {
    return 'Space';
  }

  // 2. Anime & Manga
  if (anyKeyword(fn, ['anime', 'samurai', 'cyberpunk_girl', 'girl', 'neon_girl', 'japanese', 'japan', 'otaku'])) {
    return 'Anime';
  }

  // 3. Cyberpunk & Sci-Fi
  if (anyKeyword(fn, ['cyber', 'tron', 'future', 'retrowave', 'synthwave', 'neon', 'matrix', 'hack', 'robot', 'mecha', 'tech', 'sci-fi', 'scifi', 'blade', 'digital'])) {
    return 'Cyberpunk';
  }

  // 4. Cars & Vehicles
  if (anyKeyword(fn, ['car', 'truck', 'race', 'vehicle', 'drive', 'muscle', 'road', 'porsche', 'ferrari', 'bmw', 'audi', 'auto', 'moto', 'bike', 'speed', 'wheel', 'drift'])) {
    return 'Cars & Vehicles';
  }

  // 5. City & Architecture
  if (anyKeyword(fn, ['city', 'building', 'architecture', 'bridge', 'street', 'urban', 'tower', 'skyline', 'tokyo', 'nocturne', 'house', 'marina', 'castle', 'structure', 'monument'])) {
    return 'City & Architecture';
  }

  // 6. Games & Movies & Music
  if (anyKeyword(fn, ['game', 'apex', 'octane', 'snap-hunt', 'enterprise', 'depeche', 'violator', 'green_day', 'movie', 'core_memory', 'groot', 'music', 'album', 'rock', 'song'])) {
    return 'Games & Movies';
  }

  // 7. OS Branding & Linux/Tech
  if (anyKeyword(fn, ['apple', 'arch', 'nord', 'hypr', 'adwaita', 'mojave', 'big-sur', 'miracleos', 'kath', 'nixos', 'debian', 'ubuntu', 'linux', 'fedora', 'windows', 'mac', 'os', 'kde', 'gnome'])) {
    return 'OS Branding';
  }

  // 8. Nature & Landscapes
  if (anyKeyword(fn, ['mountain', 'nature', 'forest', 'lake', 'sunset', 'landscape', 'river', 'desert', 'tree', 'field', 'ocean', 'sea', 'autumn', 'winter', 'summer', 'spring', 'island', 'deer', 'african', 'view', 'sunlight', 'sky', 'cloud', 'water', 'peak', 'hill', 'flower', 'fish', 'dew', 'horizon', 'highland', 'lofoten', 'ireland', 'italy', 'wood', 'leaf', 'valley', 'cave', 'sand', 'dune'])) {
    return 'Nature';
  }

  // 9. Abstract & Minimal
  if (anyKeyword(fn, ['abstract', 'gradient', 'pattern', 'minimal', 'wave', 'color', '3d', 'solitary', 'emergence', 'escape', 'ai-machine', 'geometry', 'shape', 'fluid', 'liquid', 'molten', 'frequency', 'lines', 'flat', 'simple', 'blur', 'light', 'dark'])) {
    return 'Abstract';
  }

  // 10. Fantasy
  if (anyKeyword(fn, ['fantasy', 'magic', 'dragon', 'sword', 'steppes', 'realm', 'myth', 'legend'])) {
    return 'Fantasy';
  }

  return 'Misc';
}

function anyKeyword(text, keywords) {
  return keywords.some(kw => text.includes(kw));
}

function formatTitle(filename) {
  const nameWithoutExt = filename.replace(/\.[^/.]+$/, '');
  
  let cleanStr = nameWithoutExt
    .replace(/[-_]+/g, ' ')
    .replace(/([a-z])([A-Z])/g, '$1 $2');

  cleanStr = cleanStr.replace(/([a-zA-Z])(\d+)/g, '$1 $2');
  cleanStr = cleanStr.replace(/(\d+)([a-zA-Z])/g, '$1 $2');

  const words = cleanStr.trim().split(/\s+/).map(word => {
    if (!word) return '';
    if (word.toLowerCase() === 'os') return 'OS';
    if (word.toLowerCase() === 'hd') return 'HD';
    return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase();
  });

  return words.join(' ');
}

function formatFileSize(bytes) {
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

function getOrientation(width, height) {
  const ratio = width / height;
  if (ratio > 1.15) return 'landscape';
  if (ratio < 0.85) return 'portrait';
  return 'square';
}

function slugify(text) {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_-]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

async function findImages(dir) {
  const entries = await fs.promises.readdir(dir, { withFileTypes: true });
  let imageFiles = [];

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);

    if (entry.isDirectory()) {
      if (entry.name.startsWith('.') || entry.name === 'node_modules' || entry.name === 'public' || entry.name === 'dist' || entry.name === 'src' || entry.name === 'scripts') {
        continue;
      }
      const subDirFiles = await findImages(fullPath);
      imageFiles = imageFiles.concat(subDirFiles);
    } else if (entry.isFile()) {
      const ext = path.extname(entry.name).toLowerCase();
      if (['.jpg', '.jpeg', '.png', '.gif'].includes(ext)) {
        imageFiles.push(fullPath);
      }
    }
  }

  return imageFiles;
}

async function processImage(fullPath) {
  const filename = path.basename(fullPath);
  const stats = await fs.promises.stat(fullPath);
  
  const category = categorizeFilename(filename);
  const title = formatTitle(filename);
  const id = slugify(`${category}-${title}`);

  // Use sharp to get metadata fast (no heavy thumbnail generation)
  const image = sharp(fullPath);
  const metadata = await image.metadata();
  const width = metadata.width || 1920;
  const height = metadata.height || 1080;
  const aspectRatio = parseFloat((width / height).toFixed(3));
  const orientation = getOrientation(width, height);

  const format = path.extname(filename).replace('.', '').toUpperCase();

  // Raw GitHub source URL
  const rawUrl = `https://raw.githubusercontent.com/mylinuxforwork/wallpaper/main/${filename}`;

  // 0MB Local Storage: On-the-fly Cloudflare CDN WebP thumbnail resizer (wsrv.nl)
  const thumbUrl = `https://wsrv.nl/?url=${encodeURIComponent(rawUrl)}&w=640&output=webp`;

  // Original CDN URL
  const cdnUrl = `https://cdn.jsdelivr.net/gh/mylinuxforwork/wallpaper@main/${filename}`;
  const cdnFallbackUrl = rawUrl;

  return {
    id,
    title,
    category,
    filename,
    path: filename,
    width,
    height,
    aspectRatio,
    orientation,
    fileSizeFormatted: formatFileSize(stats.size),
    fileSizeBytes: stats.size,
    format,
    cdnUrl,
    cdnFallbackUrl,
    thumbUrl,
    blurDataUrl: ''
  };
}

async function main() {
  console.log('🚀 Generating manifest (0MB Local Storage Mode with wsrv.nl CDN resizer)...');
  const startTime = Date.now();

  await fs.promises.mkdir(PUBLIC_DIR, { recursive: true });

  const images = await findImages(SOURCE_DIR);
  console.log(`Found ${images.length} images. Processing metadata...`);

  const manifestData = await Promise.all(images.map(processImage));

  // Sort by category then title
  manifestData.sort((a, b) => {
    if (a.category !== b.category) {
      return a.category.localeCompare(b.category);
    }
    return a.title.localeCompare(b.title);
  });

  await fs.promises.writeFile(MANIFEST_PATH, JSON.stringify(manifestData, null, 2));

  const duration = ((Date.now() - startTime) / 1000).toFixed(2);
  console.log(`✅ Done! Created manifest.json in ${duration}s with 0MB thumbnail build storage!`);
}

main().catch(err => {
  console.error('❌ Error generating manifest:', err);
  process.exit(1);
});
