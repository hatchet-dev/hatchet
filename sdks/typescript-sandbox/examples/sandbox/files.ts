import { hatchet } from '../hatchet-client';

// > Work with files
export async function prepareProject(sandboxId: string, files: Record<string, string>) {
  const sandbox = await hatchet.sandbox.connect(sandboxId);

  await sandbox.files.write(
    Object.entries(files).map(([path, data]) => ({ path: `src/${path}`, data }))
  );
  await sandbox.files.write('config/app.json', JSON.stringify({ debug: true }));

  const entries = await sandbox.files.list('src');
  const config = await sandbox.files.read('config/app.json');
  const hasLock = await sandbox.files.exists('src/package-lock.json');

  await sandbox.files.remove('src/.DS_Store');

  return { files: entries.map((e) => e.name), config: JSON.parse(config), hasLock };
}
// !!
