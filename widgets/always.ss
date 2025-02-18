x ->
  { init = x ' cmd/out x
  , subs = []
  , update = model -> _ -> model ' cmd/out model
  , view = _ -> text ""
  }
