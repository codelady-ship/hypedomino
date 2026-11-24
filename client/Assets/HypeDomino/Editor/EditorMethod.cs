using System;
using System.IO;
using UnityEditor;
using UnityEngine;

namespace HypeDomino.Editor
{
    public static class EditorMethod
    {
        public const string SucceededMarker = "Logs/editor-method-succeeded";

        public static void Run(Action method)
        {
            try
            {
                method();
                Directory.CreateDirectory(Path.GetDirectoryName(SucceededMarker));
                File.WriteAllText(SucceededMarker, string.Empty);
                EditorApplication.Exit(0);
            }
            catch (Exception failure)
            {
                Debug.LogException(failure);
                EditorApplication.Exit(1);
            }
        }
    }
}
