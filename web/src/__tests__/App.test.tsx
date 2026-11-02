import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import App from '../App'

describe('App shell', () => {
  it('should render the app name when mounted', () => {
    render(<App />)
    expect(screen.getByRole('heading', { name: 'sr-english' })).toBeInTheDocument()
  })
})
