# Angular Development Quick Reference

## Project Commands

```bash
# Install dependencies
npm install

# Development server (http://localhost:4200)
npm start

# Build for production
npm run build

# Run tests
npm test

# Serve production build locally
npm run serve:ssr:bus-schedule-lounge
```

## Project Structure

### Core (`/src/app/core`)
- **config/** - Application configuration
- **interceptors/** - HTTP interceptors (auth, error handling)
- **models/** - TypeScript interfaces and types
- **services/** - Singleton services (API, auth, state)

### Features (`/src/app/features`)
Each feature has:
- `*.component.ts` - Component logic
- `*.component.html` - Template
- `*.component.scss` - Styles

Current features:
- **auth/** - Login, authentication
- **home/** - Landing page
- **lounge/** - Lounge information & management
- **schedules/** - Bus schedule display & management

### Shared (`/src/app/shared`)
- **components/** - Reusable UI components (header, spinner, etc.)

## Adding New Features

### Generate a new component:
```bash
ng generate component features/my-feature --standalone
```

### Generate a new service:
```bash
ng generate service core/services/my-service
```

### Add a route:
Edit `src/app/app.routes.ts`:
```typescript
{ path: 'my-route', component: MyComponent }
```

## Code Patterns

### Using the API Service
```typescript
import { ApiService } from '@core/services';

constructor(private api: ApiService) {}

getData() {
  this.api.get<MyType>('/endpoint').subscribe(data => {
    // handle data
  });
}
```

### Using Models
```typescript
import { Schedule, User } from '@core/models';

schedule: Schedule = {
  id: '1',
  route: 'A-B',
  departureTime: '10:00',
  arrivalTime: '12:00',
  status: 'on-time'
};
```

### Environment Config
```typescript
import { environment } from '@environments/environment';

const apiUrl = environment.apiUrl;
```

## File Naming Conventions
- Components: `my-feature.component.ts`
- Services: `my-service.service.ts`
- Models: `my-model.ts`
- Use kebab-case for files and folders
