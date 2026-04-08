import { Button } from "./components/ui/button"

function App() {
  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <div className="space-y-4 text-center">
        <h1 className="scroll-m-20 text-4xl font-extrabold tracking-tight lg:text-5xl">
          AgendadorPlus MVP
        </h1>
        <p className="leading-7 [&:not(:first-child)]:mt-6 text-muted-foreground">
          Frontend Scaffolding with Vite, React, TypeScript + Tailwind v4 & shadcn/ui.
        </p>
        <div className="pt-4">
          <Button>Ready to Build 🚀</Button>
        </div>
      </div>
    </div>
  )
}

export default App
