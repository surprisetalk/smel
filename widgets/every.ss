ms ->
  { init = ms ' cmd/none
  , subs = [ sub/every { ms = ms, do = #tick () } ]
  , update = _ -> #tick t -> t ' cmd/out t
  , view = t -> text (text/from-int t)
  }

