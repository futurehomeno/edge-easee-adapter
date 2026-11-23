- [x] Slow start, stop, start without a mode: slow current, or the cached current without the floor
      when no slow current is configured (test first)
- [x] Normal start, stop, start without a mode: the start current (test first)
- [x] Slow start, the session ends, start without a mode: the start current (test first)
- [x] Slow start, then an explicit normal start: the start current (test first)
- [x] The session-finished observation clears the remembered mode (test first)
- [x] Issue 37 archived; issue 38 records that the mode does not survive a restart
