import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { finalize, Subscription } from 'rxjs';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.scss']
})
export class LoginComponent implements OnInit, OnDestroy {
  username = '';
  password = '';
  errorMessage = '';
  isSubmitting = false;
  private loginSub?: Subscription;

  constructor(
    private router: Router,
    private route: ActivatedRoute,
    private authService: AuthService
  ) {}

  ngOnInit(): void {
    if (this.authService.isAuthenticated()) {
      this.router.navigate(['/dashboard']);
    }
  }

  ngOnDestroy(): void {
    this.loginSub?.unsubscribe();
  }

  submit(): void {
    if (!this.username.trim() || !this.password.trim()) {
      this.errorMessage = 'Username and password are required.';
      return;
    }

    this.errorMessage = '';
    this.isSubmitting = true;

    this.loginSub?.unsubscribe();
    this.loginSub = this.authService.login(this.username.trim(), this.password.trim())
      .pipe(finalize(() => {
        this.isSubmitting = false;
      }))
      .subscribe({
        next: () => {
          const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl') || '/dashboard';
          this.router.navigateByUrl(returnUrl);
        },
        error: (err) => {
          this.errorMessage = err?.error?.error || 'Login failed. Please check your credentials.';
        }
      });
  }

  navigateHome(): void {
    this.router.navigate(['/home']);
  }

  forgotPassword(event: Event): void {
    event.preventDefault();
    // Placeholder: add a real reset flow later.
  }
}
