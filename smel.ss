{widget,sub,cmd,err,act,pipe,ui}

; widget : model => msg =>
    { init : model ' cmd msg
    , subs : model -> list (sub msg)
    , update : model -> msg -> model ' cmd msg
    , view : model -> ui msg
    }

; sub : msg =>
    #end msg
    #in (stuff -> msg)
    #key (byte -> act -> msg)
    #touch (point -> act -> msg)
    #every { ms : int, do : time -> msg }

; cmd : msg =>
    #none
    #end
    #err text
    #out stuff
    #batch (list (cmd msg))

; act :
    #up
    #down
    #press

; stuff :
    #nothing
    #scrap scrap
     #sliver bytes
    #text text
     #affix text
    #table (dict text (list scrap))
     #row (dict text scrap)
    #package bytes
     #packet bytes
    #blob bytes
     #chunk bytes
    #list (list scrap)
     #item scrap

; ui : msg =>
    #text { wrap : bool, data : text, ..style }
    #cols { data : list ui, ..style }
    #rows { data : list ui, ..style }
    #tip { label : text, data : ui, ..style }
    #button { click : msg, label : text, ..style }
    #input { input : text -> msg, label : text, ..style }
    #progress { label : text, count : float, total : float, ..style }

; style :
    { align : #left #center #stretch #right
    , border : #none #solid
    , border-color : #inherit #color color
    , font-color : #inherit #color color
    , font-size : #title #subtitle #body #note
    , font-style : #unstyled #bold #italic #strike #underline
    }
