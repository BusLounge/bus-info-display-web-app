import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-logout',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="logout-screen">
      <p>Signing you out...</p>
    </div>
  `,
  styles: [
    `
      .logout-screen {
        min-height: 100vh;
        display: grid;
        place-items: center;
        font-weight: 600;
        color: #334155;
        background: #f8fafc;
      }
    `,
  ],
})
export class LogoutComponent implements OnInit {
  constructor(
    private authService: AuthService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.authService.logout();
    this.router.navigate(['/login']);
  }
}