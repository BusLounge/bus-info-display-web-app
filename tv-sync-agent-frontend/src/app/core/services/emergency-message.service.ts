import { Injectable, signal } from '@angular/core';
import { EmergencyMessage } from '../models';

@Injectable({
  providedIn: 'root'
})
export class EmergencyMessageService {
  private activeMessage = signal<EmergencyMessage | null>(null);

  readonly activeMessage$ = this.activeMessage.asReadonly();

  createEmergencyMessage(message: Omit<EmergencyMessage, 'id'>): void {
    const newMessage: EmergencyMessage = {
      ...message,
      id: this.generateId()
    };
    this.activeMessage.set(newMessage);
  }

  updateEmergencyMessage(id: string, updates: Partial<EmergencyMessage>): void {
    const current = this.activeMessage();
    if (current && current.id === id) {
      this.activeMessage.set({ ...current, ...updates });
    }
  }

  clearEmergencyMessage(): void {
    this.activeMessage.set(null);
  }

  private generateId(): string {
    return `em_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}
