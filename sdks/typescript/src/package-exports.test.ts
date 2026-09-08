import fs from 'fs';
import path from 'path';

// The package publishes from `dist`, so the exports map is written dist-relative.
// Node ignores directory indexes once `exports` exists, so every `src/**/index.ts`
// directory needs its own entry or deep imports of it would break on publish.
describe('package.json exports', () => {
  const root = path.resolve(__dirname, '..');
  const pkg = JSON.parse(fs.readFileSync(path.join(root, 'package.json'), 'utf8'));
  const exportsMap: Record<string, unknown> = pkg.exports;

  function indexDirectories(dir: string): string[] {
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
      if (!entry.isDirectory() || entry.name === 'examples') return [];
      const full = path.join(dir, entry.name);
      const here = fs.existsSync(path.join(full, 'index.ts')) ? [full] : [];
      return [...here, ...indexDirectories(full)];
    });
  }

  it('lists the root, edge, v1 and embedded entry points', () => {
    expect(exportsMap['.']).toEqual({ types: './index.d.ts', default: './index.js' });
    expect(exportsMap['./edge']).toEqual({
      types: './edge/index.d.ts',
      default: './edge/index.js',
    });
    expect(exportsMap['./v1']).toEqual({ types: './v1/index.d.ts', default: './v1/index.js' });
    expect(exportsMap['./v1/embedded']).toEqual({
      types: './v1/embedded.d.ts',
      default: './v1/embedded.js',
    });
    expect(exportsMap['./*']).toEqual({ types: './*.d.ts', default: './*.js' });
    expect(pkg.main).toBe('index.js');
    expect(pkg.types).toBe('index.d.ts');
  });

  it('has an explicit entry for every directory with an index.ts', () => {
    const srcRoot = path.join(root, 'src');
    const missing = indexDirectories(srcRoot)
      .map((dir) => `./${path.relative(srcRoot, dir).split(path.sep).join('/')}`)
      .filter((subpath) => !exportsMap[subpath]);

    expect(missing).toEqual([]);
  });

  it('points every explicit entry at a source file that exists', () => {
    const srcRoot = path.join(root, 'src');
    const broken = Object.entries(exportsMap)
      .filter(([subpath]) => !subpath.includes('*') && subpath !== './package.json')
      .map(([, target]) => (target as { default: string }).default)
      .filter((target) => !fs.existsSync(path.join(srcRoot, target.replace(/\.js$/, '.ts'))));

    expect(broken).toEqual([]);
  });
});
