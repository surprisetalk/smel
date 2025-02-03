{ init = 
  { type = type::uniform { min = 0, max = 1 }
  , last = none
  }
, subs = [ in (#roll ()) ]
, update = model ->
    | #roll _ -> model.type
        | { ..., type = #uniform { min, max } } -> model ' (random (n -> n * (max - min) - min) |> task/perform #out)
        | _ -> model ' cmd/err "TODO"
    | #type type -> { ..model, type } ' cmd/none
    | #out n -> { ..model, last = some n } ' cmd/out n
, view = { type, last } ->
    rows []
    [ cols []
      [ text (last |> maybe/map from-int |> maybe/default "")
      , button "Roll again" (#roll ()) []
      ]
    , cols []
      [ button "Uniform" () [ disabled (type |> | #uniform _ -> true | _ -> false) ]
      , button "Normal" () [ disabled (type |> | #normal _ -> true | _ -> false) ]
      , button "Exponential" () [ disabled (type |> | #exponential _ -> true | _ -> false) ]
      , button "Poisson" () [ disabled (type |> | #poisson _ -> true | _ -> false) ]
      , button "Binomial" () [ disabled (type |> | #binomial _ -> true | _ -> false) ]
      ]
    , type |>
      | #uniform { min, max } ->
          rows []
          [ input "min" min (n -> #type (#uniform { min = n, max }))
          , input "max" max (n -> #type (#uniform { min, max = n }))
          ]
      | #normal _ -> "TODO"
      | #exponential _ -> "TODO"
      | #poisson _ -> "TODO"
      | #binomial _ -> "TODO"
    ]
}

; type :
    #uniform { min : float, max : float }
    #normal { mu : float, sigma : float }
    #exponential { lambda : float }
    #poisson { lambda : float }
    #binomial { n : int, p : float }
