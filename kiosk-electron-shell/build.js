const { execSync } = require('child_process');
const fs = require('fs-extra');
const path = require('path');

async function build() {
    try {
        console.log('--- Starting Desktop Application Build ---');

        const rootDir = path.join(__dirname, '..');
        const frontendDir = path.join(rootDir, 'frontend');
        const goDir = path.join(rootDir, 'tv-sync-agent-go');
        const shellDir = __dirname;

        const binDir = path.join(shellDir, 'bin');
        const wwwDir = path.join(shellDir, 'www');

        // 1. Clean previous builds
        console.log('\n[1/4] Cleaning previous builds...');
        fs.removeSync(binDir);
        fs.removeSync(wwwDir);
        fs.ensureDirSync(binDir);
        fs.ensureDirSync(wwwDir);

        // 2. Build Go Backend
        console.log('\n[2/4] Building Go backend...');
        const goOutPath = path.join(binDir, 'agent.exe');
        execSync(`go build -o "${goOutPath}" cmd/agent/main.go`, { cwd: goDir, stdio: 'inherit' });
        console.log('Go build successful.');

        // 3. Copy Angular Frontend
        // We assume 'npm run build' was already run or we can run it here
        console.log('\n[3/4] Building Angular frontend...');
        execSync(`npm run build`, { cwd: frontendDir, stdio: 'inherit' });
        
        console.log('Copying Angular output to www directory...');
        const angularDist = path.join(frontendDir, 'dist', 'bus-schedule-lounge', 'browser');
        fs.copySync(angularDist, wwwDir);
        console.log('Frontend copy successful.');

        // 4. Package with Electron Builder
        console.log('\n[4/4] Packaging with electron-builder...');
        execSync(`npx electron-builder --win`, { cwd: shellDir, stdio: 'inherit' });
        
        console.log('\n--- Build Complete! ---');
        console.log(`Installer is located in ${path.join(shellDir, 'dist')}`);

    } catch (error) {
        console.error('Build failed:', error.message);
        process.exit(1);
    }
}

build();
