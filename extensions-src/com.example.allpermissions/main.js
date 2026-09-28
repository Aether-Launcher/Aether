// main.js
// This script demonstrates the usage of all available Aether API permissions.

// --- UI APIs ---
// Register a sidebar page
try {
  Aether.ui.registerSidebarPage({
    id: 'all-permissions-page',
    text: 'All Permissions',
    icon: 'star', // Using a placeholder icon
    page: '/pages/all_permissions.html'
  });
  console.log('UI API: registerSidebarPage called successfully.');
} catch (error) {
  console.error('UI API: Error calling registerSidebarPage:', error);
}

// Use dialogs (demonstrating a placeholder for future dialogs)
async function demonstrateDialogs() {
  console.log('Attempting to demonstrate dialogs...');
  try {
    // NOTE: The actual Aether.dialogs.open API might not be fully implemented or might work differently.
    // This is a placeholder to show where it would be called.
    // await Aether.dialogs.open({ title: 'My Dialog', content: 'Hello!' });
    console.log('Dialogs API (placeholder): Functionality to be implemented.');
  } catch (error) {
    console.error('Dialogs API: Error during demonstration:', error);
  }
}

// --- Instance Management APIs ---
async function demonstrateInstanceApis() {
  console.log('Demonstrating Instance APIs...');
  try {
    // Query instances
    let instances;
    try {
      instances = await Aether.instances.get();
      console.log('Instance API: get() called successfully. Instances found:', instances ? instances.length : 0);
    } catch (error) {
      console.error('Instance API: Error calling get():', error);
      return; // Stop if we can't even get instances
    }

    if (instances && instances.length > 0) {
      const instanceIdToPatch = instances[0].id;
      console.log(`Instance API: Attempting to patch instance: ${instanceIdToPatch}`);
      try {
        await Aether.instances.patch(instanceIdToPatch, { version: '1.19.2' });
        console.log(`Instance API: patch() called successfully for instance: ${instanceIdToPatch}`);
      } catch (error) {
        console.error('Instance API: Error calling patch():', error);
      }
    } else {
      console.log('Instance API: No instances found to patch.');
    }
  } catch (error) {
    // This catch block would handle errors not caught by the inner try-catch
    console.error('Instance APIs: General demonstration error:', error);
  }
}

// --- Mod Loader API ---
async function demonstrateModLoaderApi() {
  console.log('Demonstrating Mod Loader API...');
  try {
    await Aether.launcher.modloader.register({
      id: 'my-custom-loader',
      name: 'My Custom Loader',
    });
    console.log('Mod Loader API: register() called successfully.');
  } catch (error) {
    console.error('Mod Loader API: Error calling register():', error);
  }
}

// --- Skin Export API ---
async function demonstrateSkinExportApi() {
  console.log('Demonstrating Skin Export API...');
  try {
    const skinDataBase64 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII='; // A 1x1 transparent pixel
    let exportedSkinPath;
    try {
      exportedSkinPath = await Aether.skin.export(skinDataBase64);
      console.log(`Skin Export API: export() called successfully. Path: ${exportedSkinPath}`);
    } catch(error) {
      console.error('Skin Export API: Error calling export():', error);
    }
  } catch (error) {
    console.error('Skin Export API: General demonstration error:', error);
  }
}

// --- Network API ---
async function demonstrateNetworkApi() {
  console.log('Demonstrating Network API...');
  try {
    let response;
    try {
      // Ensure the URL is correct and accessible
      const url = 'https://api.github.com/users/wailsapp';
      console.log(`Network API: Calling http.get() for URL: ${url}`);
      response = await Aether.http.get(url);
      console.log('Network API: http.get() received response.');
      
      // Check if response is OK before proceeding
      if (response.ok) {
        const data = await response.json();
        console.log('Network API: Received and parsed JSON data. Login:', data.login);
      } else {
        console.error('Network API: HTTP request failed with status:', response.status);
      }
    } catch (error) {
      console.error('Network API: Error during http.get() or response processing:', error);
    }
  } catch (error) {
    console.error('Network API: General demonstration error:', error);
  }
}

// --- File System API ---
async function demonstrateFsDownloadApi() {
  console.log('Demonstrating File System Download API...');
  try {
    const downloadUrl = 'https://raw.githubusercontent.com/wailsapp/wails/master/README.md';
    const savePath = 'wails_readme.md'; // Relative path within the extension's data directory
    let downloadedFilePath;
    try {
      console.log(`File System API: Calling fs.download() for URL: ${downloadUrl}`);
      downloadedFilePath = await Aether.fs.download(downloadUrl, savePath);
      console.log(`File System API: fs.download() called successfully. Path: ${downloadedFilePath}`);
    } catch (error) {
      console.error('File System API: Error calling fs.download():', error);
    }
  } catch (error) {
    console.error('File System API: General demonstration error:', error);
  }
}

// --- Execute all demonstrations ---
async function runAllDemos() {
  console.log('Starting all API demonstrations...');
  await demonstrateDialogs();
  await demonstrateInstanceApis();
  await demonstrateModLoaderApi();
  await demonstrateSkinExportApi();
  await demonstrateNetworkApi();
  await demonstrateFsDownloadApi();
  console.log('All API demonstrations initiated. Check console logs for details and errors.');
}

// Run the demonstrations when the extension starts
runAllDemos();
