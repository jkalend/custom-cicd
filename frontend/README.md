# CI/CD Agent Frontend

A modern React dashboard for managing CI/CD pipelines, built with Next.js, TypeScript, and Tailwind CSS.

## 🚀 Features

- **Modern UI**: Built with Next.js, TypeScript, and Tailwind CSS
- **Real-time Updates**: Auto-refreshing dashboard with live pipeline status
- **Pipeline Management**: Create, run, monitor, and manage CI/CD pipelines
- **Live Logs**: Real-time log viewer with automatic scrolling
- **Responsive Design**: Works great on desktop and mobile devices
- **Type Safety**: Full TypeScript support for better development experience

## 🛠️ Prerequisites

- Node.js 18+ 
- npm or yarn
- Python 3.11+ (for the backend API)

## ⚡ Quick Start

### Development Mode

1. **Start the application** (from this directory):
   ```bash
   npm install
   npm run dev
   ```

2. **Open your browser** to [http://localhost:3000](http://localhost:3000)

**Note**: The frontend includes Next.js API routes that communicate directly with the Python agent - no separate backend server needed!

### Production Build

```bash
npm run build
npm run start
```

## 📁 Project Structure

```
frontend/
├── src/
│   ├── app/                 # Next.js App Router pages & API
│   │   ├── api/             # Next.js API routes
│   │   │   ├── pipelines/   # Pipeline endpoints
│   │   │   ├── runs/        # Run endpoints  
│   │   │   └── health/      # Health check
│   │   └── page.tsx         # Main dashboard page
│   ├── components/          # Reusable React components
│   │   └── ui/              # UI components
│   │       ├── status-badge.tsx
│   │       └── log-viewer.tsx
│   ├── lib/                 # Utility functions and clients
│   │   ├── api.ts           # Frontend API client
│   │   ├── python-agent.ts  # Python agent interface
│   │   └── utils.ts         # Helper utilities
│   └── types/               # TypeScript type definitions
│       └── api.ts           # API types
├── public/                  # Static assets
├── next.config.js           # Next.js configuration
├── tailwind.config.js       # Tailwind CSS configuration
└── package.json             # Dependencies and scripts
```

## 🎯 Usage

### Creating Pipelines

1. Edit the JSON in the "Create Pipeline" section
2. Click "Create Pipeline" to save it
3. Click "Create & Run" to create and immediately execute

### Managing Pipelines

- **Run**: Start a pipeline execution
- **Details**: View pipeline configuration and step status in logs
- **Runs**: Show all runs for a specific pipeline in logs
- **Cancel**: Stop all running instances
- **Delete**: Remove the pipeline permanently

### Monitoring

- **Auto-refresh**: Dashboard updates every 5 seconds
- **Real-time logs**: View live execution logs
- **Status badges**: Visual indicators for pipeline and run states
- **Duration tracking**: See how long runs take to complete

## 🔧 Configuration

### Environment Variables

Create a `.env.local` file to customize the backend API URL:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Architecture

The application uses a hybrid architecture:

- **Frontend**: Next.js with TypeScript and Tailwind CSS
- **API Layer**: Next.js API routes (in `/src/app/api/`)
- **Backend**: Python CI/CD agent (direct communication via subprocess)

This eliminates the need for a separate Flask server while preserving the robust Python agent logic.

## 🎨 Customization

### Adding New Pages

Thanks to Next.js App Router, adding new pages is simple:

1. Create a new directory in `src/app/`
2. Add a `page.tsx` file
3. The route will be automatically available

Example:
```
src/app/settings/page.tsx → /settings
src/app/pipeline/[id]/page.tsx → /pipeline/123
```

### Styling

- **Tailwind CSS**: Utility-first CSS framework
- **Responsive Design**: Built-in mobile-first approach
- **Dark Mode**: Ready for dark mode implementation
- **Component System**: Reusable UI components

### Adding Features

The codebase is structured for easy extension:

- **API Client**: Extend `src/lib/api.ts` for new endpoints
- **Types**: Add new types in `src/types/api.ts`
- **Components**: Create reusable components in `src/components/`
- **Utils**: Add helper functions in `src/lib/utils.ts`

## 🚀 Deployment

### Vercel (Recommended)

1. Push your code to GitHub
2. Import your repository in Vercel
3. Deploy with one click

### Docker

```bash
# Build the image
docker build -t cicd-frontend .

# Run the container
docker run -p 3000:3000 cicd-frontend
```

### Static Export

```bash
npm run build
npm run export
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📝 License

This project is licensed under the MIT License.
