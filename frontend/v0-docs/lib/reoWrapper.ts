/**
 * This is a wrapper for the reodotdev library to handle the ES Module / CommonJS compatibility issues.
 * It dynamically imports the library at runtime on the client side only to avoid server-side import errors.
 */

interface ReoScriptOptions {
  clientID: string;
}

export async function loadReoScript(options: ReoScriptOptions): Promise<any> {
  // Only import on client side
  if (typeof window !== 'undefined') {
    // Use dynamic import to load the module at runtime
    try {
      // This will be imported only on the client side
      const reodotdevModule = await import('reodotdev');
      
      // Access the loadReoScript function if it exists
      if (reodotdevModule && typeof reodotdevModule.loadReoScript === 'function') {
        return reodotdevModule.loadReoScript(options);
      } else {
        throw new Error('loadReoScript function not found in reodotdev module');
      }
    } catch (error) {
      console.error('Error importing reodotdev:', error);
      throw error;
    }
  }
  
  // Return a dummy promise on server side
  return Promise.resolve({
    init: () => console.log('Reo initialization skipped on server')
  });
} 