/** Text-to-speech the study screens need; tests pass a fake. */
export interface Speech {
  hasEnglishVoice(): boolean
  speak(text: string): void
}

export function browserSpeech(): Speech {
  const synth = typeof window !== 'undefined' ? window.speechSynthesis : undefined
  // Chrome loads voices lazily; asking once at startup makes them ready before a study session opens.
  synth?.getVoices()
  return {
    hasEnglishVoice: () => !!synth && synth.getVoices().some((v) => v.lang.toLowerCase().startsWith('en')),
    speak(text) {
      if (!synth) return
      const utterance = new SpeechSynthesisUtterance(text)
      utterance.lang = 'en-GB'
      synth.cancel()
      synth.speak(utterance)
    },
  }
}
