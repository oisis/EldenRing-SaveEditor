import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'
import {ErrorBoundary} from './components/ErrorBoundary'
import {SafetyModeProvider} from './state/safetyMode'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <ErrorBoundary>
            <SafetyModeProvider>
                <App/>
            </SafetyModeProvider>
        </ErrorBoundary>
    </React.StrictMode>
)
