import { Component, OnDestroy } from '@angular/core';
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  template: '<router-outlet></router-outlet>'
})
export class App implements OnDestroy {
  // Ctrl+Shift+A asks the Electron shell to open the kiosk admin login.
  private readonly onDocumentKeydown = (event: KeyboardEvent) => {
    if (!event.ctrlKey || !event.shiftKey || event.altKey || event.metaKey || event.key.toLowerCase() !== 'a') {
      return;
    }

    const electronBridge = (window as Window & { electron?: { openAdmin?: () => void } }).electron;
    if (!electronBridge?.openAdmin) {
      return;
    }

    event.preventDefault();
    electronBridge.openAdmin();
  };

  constructor() {
    document.addEventListener('keydown', this.onDocumentKeydown);
  }

  ngOnDestroy(): void {
    document.removeEventListener('keydown', this.onDocumentKeydown);
  }
}
