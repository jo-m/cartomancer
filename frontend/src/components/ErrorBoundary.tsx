import { Component } from "react"
import type { ErrorInfo, ReactNode } from "react"
import PageContainer from "./ui/PageContainer"
import Alert from "./ui/Alert"
import Button from "./ui/Button"

interface Props {
  children: ReactNode
}

interface State {
  error: Error | null
}

/**
 * Top-level error boundary that catches unhandled rendering errors
 * and displays a themed fallback UI with a retry button.
 */
export default class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Uncaught error:", error, info.componentStack)
  }

  render() {
    if (this.state.error) {
      return (
        <PageContainer size="sm">
          <div className="flex flex-col items-center gap-4 py-16 text-center">
            <h1 className="font-heading text-2xl text-text">
              Something went wrong
            </h1>
            <Alert variant="error">{this.state.error.message}</Alert>
            <Button
              variant="secondary"
              onClick={() => this.setState({ error: null })}
            >
              Try again
            </Button>
          </div>
        </PageContainer>
      )
    }

    return this.props.children
  }
}
